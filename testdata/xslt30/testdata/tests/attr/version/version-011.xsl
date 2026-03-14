<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
 xmlns:special="http://special.uri/">

  <!-- PURPOSE: Test that exclude-result-prefix attribute can be used on any XSLT element. 
       (This is an XSLT 2.0 feature) -->
       <?spec xslt#lre-namespaces?>
  <?same-as-1.0 no?>     
  <xsl:template match="/" exclude-result-prefixes="special">
    <out>
        baccarat
    </out>
  </xsl:template>
   
</xsl:stylesheet>
