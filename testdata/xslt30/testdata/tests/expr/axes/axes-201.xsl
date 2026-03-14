<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"  
version="2.0">

<?spec fo#func-id?>
<?spec fo#func-idref?>

     <xsl:template match="/">
     <out>
         Using id():
         <xsl:apply-templates select="id('e02')"/>
         Using idref():
         [<xsl:value-of select="count(idref('e02'))"/>]
         <xsl:apply-templates select="idref('e02')/.."/>
         End
     </out>    
     </xsl:template>
     <xsl:template match="*[@name]">
         <xsl:value-of select="@name"/>
     </xsl:template>
</xsl:stylesheet>