<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test use of regex-group() within non-matching-substring (should be a zero-length string). -->
  
  <xsl:template match="/">
    <out>
      <xsl:analyze-string select="doc" regex="(M.*?y )|(h.*?d )|(l.*?e )" flags="s">
       <xsl:matching-substring>[]</xsl:matching-substring>
       <xsl:non-matching-substring>{<xsl:value-of select="sum(for $i in 1 to 10 return count(regex-group($i)))"/>}</xsl:non-matching-substring>
      </xsl:analyze-string>      
    </out>
  </xsl:template>

</xsl:stylesheet>