<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:string accessor with copied nodes as argument -->

   <xslt:template match="/">
	
	     <xslt:variable name="var1">
	        <xslt:copy-of select="/"/>
      </xslt:variable>
  
	     <out>
         <t1>
            <xslt:value-of select="string($var1/doc/processing-instruction())"/>
         </t1>
         <t2>
            <xslt:value-of select="string($var1/doc/comment())"/>
         </t2>
         <t3>
            <xslt:value-of select="string($var1/doc/text()[1])"/>
         </t3>
         <t4>
            <xslt:value-of select="string($var1/doc/element())"/>
         </t4>
         <t5>
            <xslt:value-of select="string($var1/doc/attribute())"/>
         </t5>
         <t6>
            <xslt:value-of select="string($var1)"/>
         </t6>
         <t7>
            <xslt:for-each select="$var1/doc/namespace::*[name(.)='mynamespace']">
				           <xslt:value-of select="string(.)"/>
			         </xslt:for-each>
         </t7>
      </out>
   </xslt:template>
</xslt:transform>
