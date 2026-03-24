<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"  version="2.0"
			    >
<?spec xslt#named-templates?>
<!-- BUG: Saxon 5.3.1/003 -->
<!-- DESCRIPTION: Under certain conditions passing a parameter in xsl:with-param that is
    a node-set-expression containing a variable reference will fail when the parameter value
    is referenced. -->

<xsl:template match="/">
		<xsl:call-template name="TestFunction"/>
</xsl:template>

<!-- ~~~~~~~~~~~~~~~   TestFunction   ~~~~~~~~~~~~~~~~ -->
<xsl:template name="TestFunction">

	<xsl:variable name="tmp-derive-from"><top/>
	</xsl:variable>

	<xsl:variable name="super" select="//item"/>
				<xsl:call-template name="GetElem">
					<xsl:with-param name="element-set" select="$super/*"/>
				</xsl:call-template>

</xsl:template>

<!-- ~~~~~~~~~~~~~~~   GetElem   ~~~~~~~~~~~~~~~~ -->
<xsl:template name="GetElem">
	<xsl:param name="element-set"/>
	<xsl:copy-of select="$element-set[@name]"/>
</xsl:template>				

</xsl:stylesheet>
