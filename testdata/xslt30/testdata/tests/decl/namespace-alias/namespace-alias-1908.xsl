<xsl:transform 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:a="http://a/com/" 
  version="2.0">
  
  <!-- test use of result-prefix="#default" when there is no default namespace -->
  <?spec xslt#namespace-aliasing?>

<xsl:namespace-alias stylesheet-prefix="a" result-prefix="#default"/>

<xsl:template match="/">
  <a:foo a:bar="baz"/>
</xsl:template>

</xsl:transform>