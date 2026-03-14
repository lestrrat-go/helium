<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-numeric-divide?>
<!-- PURPOSE:  Test integer division (XPath 2.0) -->
<xsl:template match="doc">
<out>
<a val="{2 idiv 2}"/>;
<a val="{4 idiv 2}"/>;
<a val="{0 idiv 2}"/>;
<a val="{10 idiv 3}"/>;
</out>
</xsl:template>
</xsl:stylesheet>