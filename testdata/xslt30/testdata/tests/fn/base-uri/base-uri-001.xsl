<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                xml:base="http://www.baseuri.exmpl/tests/"
                version="2.0">
<!-- Purpose: Testcase with fn:resolve-uri($relative) where $relative is an absolute URI. 
  				The result is $relative unchanged. -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:variable name="var" select="/doc" as="xs:string"/>

   <xslt:template match="/doc">
      <output xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="resolve-uri('http://www.baseuri.exmpl/001.xml')"/>
         </out>
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="resolve-uri($var)"/>
         </out>
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="resolve-uri(string(.))"/>
         </out>
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="resolve-uri(subsequence(('http://www.baseuri.exmpl/001.xml', '', .),1,1))"/>
         </out>
      </output>
   </xslt:template>
</xslt:transform>
