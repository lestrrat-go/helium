<?xml version="1.0" encoding="iso-8859-1"?>
<xsl:transform
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 version="2.0"
>

<?spec ser#html?>
<!-- BUG: Saxon 5.3/007 -->
<!-- DESCRIPTION: The HTML output method occasionally applies URL escaping to an attribute that is not
    a URL, for example (with the Microsoft JVM) it does so with &lt;h2 style="clear: all"&gt;. -->

<xsl:output method="html" encoding="iso-8859-1"/>
<xsl:strip-space elements="*"/>


<xsl:template match="/">
<html><body><h2 style="clear: all"></h2></body></html>
</xsl:template>

<!--å-->

</xsl:transform>
