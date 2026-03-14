<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- PURPOSE: test non-capturing-groups allowed from XSLT 3.0 only -->
  
  <xsl:param name="in" select="'ABCDEFGH'"/>

  <xsl:template match="/">
    <out>
      <xsl:analyze-string select="$in" regex="(?:AB)(CD)(?:EF)(GH)">
       <xsl:matching-substring><m g1="{regex-group(1)}" g2="{regex-group(2)}" g3="{regex-group(3)}" g4="{regex-group(4)}"/></xsl:matching-substring>
       <xsl:non-matching-substring><wrong/></xsl:non-matching-substring>
      </xsl:analyze-string>      
    </out>
  </xsl:template>
  

</xsl:stylesheet>