<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:x = "http://xxx.com/"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="x xs">

<!-- Promotion of xs:anyURI to xs:string -->

<xsl:variable name="data"><a xmlns="http://a.com/"/></xsl:variable>

<xsl:function name="x:test" as="xs:boolean" >
    <xsl:param name="in" as="xs:string"/>
    <xsl:sequence select="x:isURI($in)"/>
</xsl:function>

<xsl:function name="x:isURI" as="xs:boolean">
  <xsl:param name="n" as="item()"/>
  <xsl:sequence  select="$n instance of xs:anyURI"/>
</xsl:function>

  <xsl:template match="/">
      <out><xsl:value-of select="x:test(namespace-uri($data/a))"/></out>
  </xsl:template>


</xsl:stylesheet>
