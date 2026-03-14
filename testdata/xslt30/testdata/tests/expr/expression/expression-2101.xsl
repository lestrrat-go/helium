<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xslt#generate-id?>

<xsl:output method="text"/>
<xsl:template match="doc">
<z>
    <!-- BUG in 7.6.5 and earlier - wrong optimization causes different results being output -->
    [<xsl:value-of select="generate-id(.) = generate-id(namespace::*[1])"/>]
    [<xsl:value-of select="generate-id(namespace::*[1]) = generate-id(.)"/>]

    <!-- BUG 2 - colon (':') is not an alphanumeric character -->
    <xsl:value-of select="matches(generate-id(@*[1]), '[A-Za-z][A-Za-z0-9]*')"/>;
    <xsl:value-of select="matches(generate-id(namespace::*[1]), '[A-Za-z][A-Za-z0-9]*')"/>;
</z>
</xsl:template>

</xsl:stylesheet>

