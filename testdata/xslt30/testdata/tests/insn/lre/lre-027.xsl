<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:one="http://one.com"
    xmlns:two="http://two.com"
    xmlns:three="http://three.com"
    xmlns:four="http://four.com"
    exclude-result-prefixes="one">

<?spec xslt#lre-namespaces?>
    <!-- Purpose: Test that exclude-result-prefixes is cumulative. -->

<xsl:template name="main">
  <out xsl:exclude-result-prefixes="two three"/>
</xsl:template>

</xsl:stylesheet>
