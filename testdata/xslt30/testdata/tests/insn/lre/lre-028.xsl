<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:one="http://one.com"
    xmlns:two="http://two.com"
    xmlns:three="http://three.com"
    xmlns:four="http://four.com"
    exclude-result-prefixes="one two three">

<?spec xslt#lre-namespaces?>
    <!-- Purpose: Test that exclude-result-prefixes applies to URIs, not to prefixes. -->

<xsl:template name="main"
    xmlns:one="http://eins.com"
    xmlns:two="http://zwei.com"
    xmlns:three="http://drei.com"
    xmlns:four="http://vier.com"
    xmlns:un="http://one.com"
    xmlns:deux="http://two.com"
    xmlns:trois="http://three.com"
    xmlns:quatre="http://four.com"
    >
  <out xsl:exclude-result-prefixes="two three"/>
</xsl:template>
    
    <xsl:variable name="expected">
        <out xmlns:one="http://eins.com"
            xmlns:four="http://vier.com"
            xmlns:quatre="http://four.com"/>
    </xsl:variable>

</xsl:stylesheet>
