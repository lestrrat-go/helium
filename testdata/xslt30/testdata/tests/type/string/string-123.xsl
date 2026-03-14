<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-string-join?>
<!-- PURPOSE:   Test string-join() -->

<xsl:template match="doc">
<out>
    <z><xsl:value-of select="string-join(a, sep)"/></z>;
    <z><xsl:value-of select="string-join(a[1], sep)"/></z>;   
    <z><xsl:value-of select="string-join(a[0], sep)"/></z>;  
    <z><xsl:value-of select="string-join(a, '')"/></z>;
    <z><xsl:value-of select="string-join('', 'xxxxx')"/></z>;                
</out>
</xsl:template>
</xsl:stylesheet>