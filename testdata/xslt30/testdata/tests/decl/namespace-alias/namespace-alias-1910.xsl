<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:a="http://a/com/"
  xmlns:b="http://b/com/" version="2.0">

  <!-- xsl:namespace-alias used to swap two namespaces -->
  <?spec xslt#namespace-aliasing?>

  <xsl:namespace-alias stylesheet-prefix="a" result-prefix="b"/>
  <xsl:namespace-alias stylesheet-prefix="b" result-prefix="a"/>

  <xsl:template match="/">
    <a:foo a:bar="baz" b:bar="zoo"/>
  </xsl:template>

</xsl:transform>
