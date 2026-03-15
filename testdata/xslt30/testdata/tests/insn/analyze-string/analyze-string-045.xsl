<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test xsl:analyze-string applied to an empty sequence - error in 2.0, treated as "" in 2.1. -->
  
  <xsl:param name="in" select="()"/>

  <xsl:template match="/">
    <out>
      <xsl:analyze-string select="$in" regex="[A-Z]">
       <xsl:matching-substring><m/></xsl:matching-substring>
       <xsl:non-matching-substring><n/></xsl:non-matching-substring>
      </xsl:analyze-string>      
    </out>
  </xsl:template>
  

</xsl:stylesheet>