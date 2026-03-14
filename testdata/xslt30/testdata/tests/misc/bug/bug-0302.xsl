<xsl:transform
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 version="2.0"
 exclude-result-prefixes="#default" xmlns="http://abc.uri/" 
>

<?spec xslt#lre-namespaces?>
<!-- BUG: Saxon 5.3/003 -->
<!-- DESCRIPTION: The attribute <b>[xsl:]exclude-result-prefixes="#default"</b> has
    no effect. -->
    
<xsl:output method="xml" encoding="iso-8859-1"/>
<xsl:strip-space elements="*"/>


<xsl:template match="/" xmlns:z="http://z.uri/">
<xsl:variable name="tmp">
<z:a a-foo="a-bar" />
<z:b b-foo="b-bar"/>
<c c-foo="c-bar" />
</xsl:variable>
<out changed="2004-04-06">
<xsl:copy-of select="$tmp"/>
</out>
</xsl:template>



</xsl:transform>
