<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet exclude-result-prefixes="xs" version="2.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <xsl:template name="xsl:initial-template">
    <out>
      <xsl:analyze-string select="'aaabbbccc'" regex="(a*)(b*)(c*)">
        <xsl:matching-substring>
           <xsl:variable name="g" select="regex-group#1"/>
           <xsl:analyze-string select="'111222333'" regex="(1*)(2*)(3*)">
             <xsl:matching-substring>
               <xsl:value-of select="$g(2)"/>
             </xsl:matching-substring>
           </xsl:analyze-string>
        </xsl:matching-substring>
      </xsl:analyze-string>
    </out>
  </xsl:template>
  
</xsl:stylesheet>