<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
               version="2.0"
>

<?spec xslt#copying?>
<!-- Tests elimination of a bug that was in the original SAX distribution, whereby a namespace
     declared on a parent element was used in an attribute of a child element before being
     redeclared -->

<xsl:strip-space elements="*"/>

<xsl:template match="*">
  <xsl:text>[</xsl:text>
  <diag name="{local-name()}" namespace-uri="{namespace-uri()}"/>
  <xsl:text>]&#x0a;</xsl:text>
  <xsl:element namespace="{namespace-uri()}" name="{local-name()}">
    <xsl:apply-templates select="@*|node()"/>
  </xsl:element>
  <xsl:text>;&#x0a;</xsl:text>
</xsl:template>

<xsl:template match="@*">
	<xsl:copy/>
</xsl:template>

<xsl:template match="/">
<out>
	<xsl:apply-templates select="node()" />
</out>    
</xsl:template>

</xsl:transform>
