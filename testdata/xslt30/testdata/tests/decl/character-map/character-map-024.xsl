<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                >
<?spec xslt#character-maps?>

<!-- test character maps with text output method -->

<xsl:output  omit-xml-declaration="yes"  method="text" use-character-maps="maple"/>

<xsl:character-map name="maple">

<xsl:output-character character="+" string="plus" />

</xsl:character-map>



</xsl:stylesheet>
