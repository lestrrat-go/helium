<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  exclude-result-prefixes="xs map">

  <xsl:template match="/">
    <xsl:variable name="results"
                  select="transform(
                            map {
                              'stylesheet-location':'transform-009a.xsl',
                              'source-node':.,
                              'delivery-format':'serialized'
                            }
                          )"/>

    <!-- Pass through already-serialized principal output document as text -->
    <xsl:result-document method="text">
      <xsl:sequence select="$results?output"/>
    </xsl:result-document>

    <!-- Concatenate already-serialized secondary output documents -->
    <xsl:result-document href="secondary-outputs.xml" method="text">
      <xsl:text>&lt;secondary-outputs></xsl:text>
      <xsl:sequence select="map:keys($results)[not(. eq 'output')] ! map:get($results, .)"/>
      <xsl:text>&lt;/secondary-outputs></xsl:text>
    </xsl:result-document>

  </xsl:template>

</xsl:stylesheet>
