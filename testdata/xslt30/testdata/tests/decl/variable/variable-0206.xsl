<xsl:stylesheet version="2.0" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:f="http://fxsl.sf.net/"
xmlns:test="my:test"
  exclude-result-prefixes="f xs test"
>
  <!-- test type error on tunnel parameter -->
  <?spec xslt#tunnel-params?>
<?error ?>
  <?same-as-1.0 no?>
  
  <xsl:template match="/">
  <out>
   <xsl:apply-templates>
     <xsl:with-param name="t1" select="17" tunnel="yes"/>
     <xsl:with-param name="t2" select="18" tunnel="yes"/>
   </xsl:apply-templates>
  </out>
  </xsl:template>
  
  <xsl:template match="b">
    <xsl:param name="t2" select="18" tunnel="yes"/>
    <b>
      <xsl:apply-templates>
        <xsl:with-param name="t2" select="28" tunnel="yes"/>
        <xsl:with-param name="t3" select="38" tunnel="yes"/>
      </xsl:apply-templates>
    </b>
    <old><xsl:value-of select="$t2"/></old>
  </xsl:template>
  
  <xsl:template match="d">
    <xsl:param name="t1" tunnel="yes" as="node()"/>
    <xsl:param name="t2" tunnel="yes" select="-1" required="no"/>
    <xsl:param name="t3" tunnel="yes" select="-1"/>        
    <d t1="{$t1}" t2="{$t2}" t3="{$t3}"/>
  </xsl:template>
    

</xsl:stylesheet>
