<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:pre="http://example.com"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="pre xs">

<!-- test that result of local-name-from-QName is an NCName -->
<?spec fo#func-local-name-from-QName?>

<xsl:template match="/">
    <xsl:variable name="x"><pre:n/></xsl:variable>
    <z answer="{local-name-from-QName(node-name($x/*)) instance of xs:NCName}"/>
</xsl:template>
</xsl:stylesheet>