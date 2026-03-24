<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>


<?spec xpath#id-arithmetic?>
<!-- BUG: Not actually a bug. Correct answer is 1000000000000 -->
<?same-as-1.0 no?>

<xsl:template match="/">
<out>
    <xsl:apply-templates select="//a2"/>
</out>
</xsl:template>

<xsl:template match="a2">
    <xsl:value-of select="*******"/>
</xsl:template>


</xsl:stylesheet>



