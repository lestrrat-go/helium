<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns="http://www.abc.com"
                xmlns:b="http://www.def.com"
                exclude-result-prefixes="b #default">

<?spec xslt#lre-namespaces?>
<!-- test that exclude-result-prefixes is ignored if the prefix is actually used -->
<!-- BUG: Saxon 5.3.1/006 -->
<!-- DESCRIPTION: When a namespace is excluded from the result tree using xsl:exclude-result-prefixes,
    but is then used on a literal result element, the request to exclude it should be
    ignored, to ensure that the result is well-formed XML following the Namespace rules.
    Currently the request to exclude the namespace is ignored if the namespace is used on
    the element containing the xsl:exclude-result-prefixes attribute, but not if it is used
    on an inner element. -->

<xsl:template match="/">
<out>
<foo/>
<b:foo/>
</out>
</xsl:template>
</xsl:stylesheet>
