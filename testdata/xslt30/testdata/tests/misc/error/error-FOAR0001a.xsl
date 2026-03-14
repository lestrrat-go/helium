<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE:  Test integer division by zero (XPath 2.0) -->
<?spec fo#func-numeric-integer-divide?><?error FOAR0001?>

<xsl:template match="doc">
<out>
<a val="{2 idiv 2}"/>;
<a val="{4 idiv 2}"/>;
<a val="{0 idiv 2}"/>;
<a val="{10 idiv 3}"/>;
<a val="{10 idiv 0}"/>;
</out>
</xsl:template>
</xsl:stylesheet>