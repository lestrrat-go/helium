<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test arithmetic between subtypes of integer. -->
  <?spec xpath#id-arithmetic?>
  <?spec fo#casting-to-derived-types?>
  
  <xsl:template match="/">
    <out>;
      <t><xsl:value-of select="xs:integer('100') * xs:decimal('100.00')"/></t>;
      <t><xsl:value-of select="xs:long('100') * xs:int('100')"/></t>;
      <t><xsl:value-of select="xs:short('-500') + xs:long(+500)"/></t>;      
      <f><xsl:value-of select="xs:short(-500) - xs:positiveInteger(+500)"/></f>;                                       
      <t><xsl:value-of select="xs:integer('100') * 20e2"/></t>;
      <t><xsl:value-of select="xs:long('100') * xs:decimal('100')"/></t>;
      <t><xsl:value-of select="xs:short('-500') div xs:double(+500)"/></t>;      
      <f><xsl:value-of select="xs:float(-500) - xs:positiveInteger(+500)"/></f>;  
    </out> 
  </xsl:template>
  

</xsl:stylesheet>