<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<!-- PURPOSE:  Test trace() function with a singleton and with an empty sequence -->
<?spec fo#func-trace?>

<xsl:param name="p" select="13"/>
<xsl:param name="q" select="'London'"/>
<xsl:param name="r" select="()"/>


<xsl:template match="doc">
<out>
<a val="{trace($p, '$p')}"/>;
<a val="{trace($q, '$q')}"/>;
<a val="{trace($r, '$r')}"/>;
</out>
</xsl:template>
</xsl:stylesheet>