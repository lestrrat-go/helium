<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test flag "q" allowed in 2.1 only -->
  
  <xsl:param name="in" select="'A.A.A.A.A'"/>

  <xsl:template match="/">
    <out>
      <xsl:analyze-string select="$in" regex="." flags="q">
       <xsl:matching-substring><m val="{.}"/></xsl:matching-substring>
       <xsl:non-matching-substring><n val="{.}"/></xsl:non-matching-substring>
      </xsl:analyze-string>      
    </out>
  </xsl:template>
  

</xsl:stylesheet>