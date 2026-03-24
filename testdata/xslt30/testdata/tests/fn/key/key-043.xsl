<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:x="http://namespaces.ogbuji.net/articles" exclude-result-prefixes="x">

  <!-- Purpose: Test combination of key() and document() reading from stylesheet. -->
  <!-- Elaboration: "Look-up table 1.6 is worth a close look because it uses an advanced XSLT
    technique. It builds up the lookup-table right in the stylesheet, using a distinct namespace.
    You can see the x:ns-to-binding elements right below the key. If you are familiar with keys,
    you are aware that they define indices that will be built on the nodes in the original source
    document that match the pattern in the match attribute. What is not as well known is that
    every time an additional source document is loaded with the XSLT document() function, all keys
    are applied to it as well. The xsl:variable...uses a special form of document() call to load
    the stylesheet itself as an additional source document. Thus the nodes in the stylesheet that
    match the ns-to-binding are indexed. This is a very useful technique for setting up a look-up
    table without having to hack at the source document or depend on an additional file." -->


  <!-- Lookup table 1.6: WSDL binding types -->
  <xsl:key name="ns-to-binding" match="x:ns-to-binding" use="@binding"/>
  <x:ns-to-binding uri="http://schemas.xmlsoap.org/wsdl/soap/" binding="SOAP"/>
  <x:ns-to-binding uri="http://schemas.xmlsoap.org/wsdl/mime/" binding="MIME"/>
  <x:ns-to-binding uri="http://schemas.xmlsoap.org/wsdl/http/" binding="HTTP"/>

  <xsl:template match="doc">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>

  <xsl:template match="bind">
    <bound>
      <xsl:variable name="lookup" select="."/>
      <xsl:value-of select="$lookup"/>
      <xsl:text>- </xsl:text>
      <xsl:for-each select="document('')">
        <!-- Switch context so key reads from stylesheet -->
        <xsl:value-of select="key('ns-to-binding',$lookup)/@uri"/>
      </xsl:for-each>
    </bound>
  </xsl:template>
  
  <xsl:template match="text()"/>

</xsl:stylesheet>
