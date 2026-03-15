<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test xsl:analyze-string using groups. -->
  
  <xsl:template match="/">
    <out>
      <xsl:variable name="v">
        <xsl:analyze-string select="doc" regex="(M.*?y )|(h.*?d )|(l.*?e )" flags="s">
         <xsl:matching-substring>[<xsl:value-of select="regex-group(0)"/>(<xsl:value-of select="regex-group(2)"/>)]</xsl:matching-substring>
         <xsl:non-matching-substring><xsl:value-of select="."/></xsl:non-matching-substring>
        </xsl:analyze-string> 
      </xsl:variable>
      <xsl:value-of select="translate($v, '&#xa;', '|')"/>
    </out>
  </xsl:template>

</xsl:stylesheet>