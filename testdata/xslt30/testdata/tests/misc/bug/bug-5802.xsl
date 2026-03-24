<xsl:transform
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 version="2.0"
>

<?spec xpath#id-path-expressions?>

<!-- ./(a,b) should sort (a,b) into document order -->
<?same-as-1.0 no?>

<xsl:template match="/">
<out>
  <xsl:copy-of select="./ ( //c[@id='33'], //c[@id='11'])"/>
  <xsl:copy-of select="( //c[@id='33'], //c[@id='11']) /."/>
</out>  
</xsl:template>



</xsl:transform>
