<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                xml:base="http://www.baseuri.exmpl/tests/"
                version="2.0">
<!-- Purpose: Testcase with fn:resolve-uri($relative) where $relative is a relative URI. 
  				$relative is resolved agains the base-uri from the static context. $relative 
  				value is a literal string, node, variable and subsequence.-->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:variable name="var" select="/doc/str1/@att"/>

   <xslt:template match="/doc">
      <output xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="resolve-uri('002.xml')"/>
         </out>
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="resolve-uri($var)"/>
         </out>
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="resolve-uri(string(str1/@att))"/>
         </out>
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="resolve-uri(subsequence(('002.xml', '', .),1,1))"/>
         </out>
      </output>
   </xslt:template>
</xslt:transform>
