<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test xs:boolean(). -->
  <?spec fo#casting-boolean?>

  <xsl:template match="/">
    <out>;
      <f><xsl:value-of select="xs:boolean('0')"/></f>;
      <t><xsl:value-of select="xs:boolean('1')"/></t>;
      <t><xsl:value-of select="xs:boolean('true')"/></t>;
      <f><xsl:value-of select="xs:boolean('false')"/></f>;
      <f><xsl:value-of select="xs:boolean(0)"/></f>;
      <t><xsl:value-of select="xs:boolean(1)"/></t>;
      <e><xsl:value-of select="if (2 castable as xs:boolean) then xs:boolean(2) else 'error'"/></e>; 
      <e><xsl:value-of select="if ('TRUE' castable as xs:boolean) then xs:boolean('TRUE') else 'error'"/></e>                                           
    </out>
  </xsl:template>

</xsl:stylesheet>