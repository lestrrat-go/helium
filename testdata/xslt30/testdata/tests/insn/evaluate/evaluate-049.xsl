<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:my="http://www.example.com/my-functions" exclude-result-prefixes="my">

  <!-- PURPOSE test xsl:evaluate uses the default collation -->


  <xsl:param name="p">"XYZ" eq "xyz"</xsl:param>
  
  <xsl:template name="xsl:initial-template" default-collation="http://www.w3.org/2013/collation/UCA?strength=secondary">
    <out>
      <xsl:evaluate xpath="$p"/>
    </out>
  </xsl:template>

 
</xsl:stylesheet>
