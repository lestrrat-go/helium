<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:string accessor with atomic value as argument -->

   <xslt:template match="/">
      <out>
         <xslt:variable name="temp1" select="xs:string('test string')"/>
         <xslt:variable name="temp2" select="xs:boolean('1')"/>
         <xslt:variable name="temp3" select="xs:float('1E4')"/>
         <xslt:variable name="temp4" select="xs:dateTime('1999-08-22T10:05:06')"/>
         <t1>
            <xslt:value-of select="string('test string')"/>
         </t1>
         <t2>
            <xslt:value-of select="string($temp1)"/>
         </t2>
         <t3>
            <xslt:value-of select="string(subsequence($temp1, 1))"/>
         </t3>
         <t1>
            <xslt:value-of select="string(xs:boolean('1'))"/>
         </t1>
         <t2>
            <xslt:value-of select="string($temp2)"/>
         </t2>
         <t3>
            <xslt:value-of select="string(subsequence($temp2, 1))"/>
         </t3>
         <t1>
            <xslt:value-of select="string(xs:dateTime('1999-08-22T10:05:06'))"/>
         </t1>
         <t2>
            <xslt:value-of select="string($temp4)"/>
         </t2>
         <t3>
            <xslt:value-of select="string(subsequence($temp4, 1))"/>
         </t3>
      </out>
   </xslt:template>
</xslt:transform>
