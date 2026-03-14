<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://www.mytest.net"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with global xsl:variable that has @as="processing-instruction()*" and 
  				@select contains: -one processing-instruction node from input file, 
  				a sequence of processing-instruction nodes from input file, an empty sequence 
  				obtained at run time, (), processing-instruction nodes from a temporary tree  -->

   <xslt:variable name="temporary-tree">
      <a att="one">
         <a att="two">hello</a>
         <b att="three">hello</b>
      </a>
   </xslt:variable>

   <xslt:variable name="var1" select="/doc/processing-instruction()" as="processing-instruction()*"/>

   <xslt:variable name="var2" select="/doc//processing-instruction()" as="processing-instruction()*"/>

   <xslt:variable name="var3" select="/doc/item/processing-instruction()"
      as="processing-instruction()*"/>

   <xslt:variable name="var4" select="()" as="processing-instruction()*"/>

   <xslt:variable name="var5" select="$temporary-tree//processing-instruction()"
      as="processing-instruction()*"/>

   <xslt:template match="/doc">
      <out>
         <e1>
            <xslt:value-of select="$var1 instance of processing-instruction()*"/>
            <xslt:value-of select="$var1 instance of processing-instruction()"/>
         </e1>
         <e2>
            <xslt:value-of select="$var2 instance of processing-instruction()*"/>
            <xslt:value-of select="$var2 instance of processing-instruction()"/>
         </e2>
         <e3>
            <xslt:value-of select="$var3 instance of processing-instruction()*"/>
            <xslt:value-of select="$var3 instance of processing-instruction()"/>
         </e3>
         <e4>
            <xslt:value-of select="$var4 instance of processing-instruction()*"/>
            <xslt:value-of select="$var4 instance of processing-instruction()"/>
         </e4>
         <e5>
            <xslt:value-of select="$var5 instance of processing-instruction()*"/>
            <xslt:value-of select="$var5 instance of processing-instruction()"/>
         </e5>
      </out>
   </xslt:template>
</xslt:transform>
