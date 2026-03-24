<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Testcase with fn:resolve-uri($relative, $base) where $relative is an absolute URI. 
  				The result is $relative unchanged. $relative and $base values are a literal strings, 
  				nodes, variables and subsequences.-->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:variable name="var1" select="/doc/str1" as="xs:string"/>

   <xslt:variable name="var2" select="/doc/str2" as="xs:string"/>

   <xslt:template match="/doc">
      <output xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="resolve-uri('http://www.baseuri.exmpl/001.xml', $var2)"/>
         </out>
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="resolve-uri($var1, $var2)"/>
         </out>
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="resolve-uri(string(str1), 'http://www.one.exmpl/')"/>
         </out>
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="resolve-uri(subsequence(('http://www.baseuri.exmpl/001.xml', '', .),1,1), string(str2))"/>
         </out>
      </output>
   </xslt:template>
</xslt:transform>
