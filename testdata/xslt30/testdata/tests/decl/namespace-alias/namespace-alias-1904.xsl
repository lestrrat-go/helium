<xsl:transform 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <!-- test use of stylesheet-prefix="#default" when there is no default namespace -->
  <?spec xslt#namespace-aliasing?>

<xsl:namespace-alias stylesheet-prefix="#default" result-prefix="xsl"/>

<xsl:template match="/">
  <stylesheet proc="bah">boo</stylesheet>
</xsl:template>

</xsl:transform>