<xsl:stylesheet version="2.0" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:f="http://fxsl.sf.net/"
xmlns:test="my:test"
  exclude-result-prefixes="f xs test"
>

<!-- error - params to apply-templates have wrong type -->
  <?spec xslt#with-param?>
<?error ?>
<?same-as-1.0 no?>

  <xsl:output omit-xml-declaration="yes"/>
  
  <xsl:template match="/">
   <xsl:apply-templates select="document('')/*/test:*[1]">
     <xsl:with-param name="arg1" select="'false'"/>
     <xsl:with-param name="arg2" select="1"/>
   </xsl:apply-templates>
  </xsl:template>
  
  <test:test/>
  <xsl:template name="And" match="test:*">
    <xsl:param name="arg1" as="xs:boolean"/>
    <xsl:param name="arg2"  as="xs:boolean"/>
         <xsl:value-of 
         select="xs:boolean(xs:boolean($arg1) and xs:boolean($arg2))"/>
  </xsl:template>

</xsl:stylesheet>
