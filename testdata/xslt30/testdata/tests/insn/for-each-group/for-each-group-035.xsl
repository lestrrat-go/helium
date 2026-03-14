<?spec fo#func-distinct-values?>
<xsl:transform 
  version="2.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  
  <!-- test that distinct-values works on a heterogeneous sequence -->
  <?category distinct-values?>
  
<xsl:template match="/">
<chapter>
    <xsl:value-of select="distinct-values(('a', 2, 4, number('NaN'), 'a', 
    xs:date('2003-02-03'), xs:date('2003-02-03'), xs:date('2003-02-04'), 4))" separator=" "/>
</chapter>
</xsl:template>
	
</xsl:transform>