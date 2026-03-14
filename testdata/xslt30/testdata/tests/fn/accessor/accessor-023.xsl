<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:data accessor with a sequence of both atomic and node types -->

   <xslt:template match="doc">
      <out>
         <xslt:variable name="var1" select="processing-instruction()"/>
         <xslt:variable name="var2" select="comment()"/>
         <xslt:variable name="var3" select="text()[1]"/>
         <xslt:variable name="temp1" select="xs:string('test_string')"/>
         <xslt:variable name="temp2" select="xs:boolean('1')"/>
         <xslt:variable name="temp3" select="xs:float('1E4')"/>
         <xslt:variable name="mix" select="($var1, $var2, $var3, $temp1, $temp2, $temp3)"/>
         <t1>
            <xslt:value-of select="data(($var1, $var2, $var3, $temp1, $temp2, $temp3))"/>
         </t1>
         <t2>
            <xslt:value-of select="data($mix)"/>
         </t2>
      </out>
   </xslt:template>
</xslt:transform>
