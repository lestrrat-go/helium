<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<!-- PURPOSE:  Type error in equality comparison -->
<?spec xpath#errors?><?error XPTY0004?>

<xsl:template match="doc">
<out>
<a val="{string(3 = '2')}"/>;
</out>
</xsl:template>
</xsl:stylesheet>