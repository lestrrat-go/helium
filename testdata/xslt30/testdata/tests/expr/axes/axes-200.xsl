<xsl:stylesheet version="2.0" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>

<?spec fo#func-base-uri?>
<?same-as-1.0 no?>

   <xsl:template match="/">
    <z><xsl:value-of select="base-uri(/doc/e)"/></z>

   </xsl:template>

</xsl:stylesheet>
