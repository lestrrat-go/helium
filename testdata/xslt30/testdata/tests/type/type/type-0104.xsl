<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test subtype relationships for subtypes of integer. -->
<?spec fo#casting-to-derived-types?>
<?spec xpath#id-instance-of?>
  
  <xsl:template match="/">
    <out>;
      <t><xsl:value-of select="xs:integer('1000000000000000') instance of xs:decimal"/></t>;
      <t><xsl:value-of select="xs:long('10000000000000000') instance of xs:integer"/></t>;
      <t><xsl:value-of select="xs:int('-100000000') instance of xs:long"/></t>;      
      <t><xsl:value-of select="xs:short('-10000') instance of xs:integer"/></t>;
      <t><xsl:value-of select="xs:byte('100') instance of xs:long"/></t>;
      <t><xsl:value-of select="xs:unsignedLong('10000000000000000') instance of xs:decimal"/></t>;
      <f><xsl:value-of select="xs:unsignedInt('+10000000') instance of xs:long"/></f>;
      <f><xsl:value-of select="xs:unsignedShort('65535') instance of xs:byte"/></f>;
      <t><xsl:value-of select="xs:unsignedByte('255') instance of xs:unsignedInt"/></t>;
      <f><xsl:value-of select="xs:nonNegativeInteger('-0') instance of xs:dateTime"/></f>;
      <t><xsl:value-of select="xs:nonPositiveInteger('0') instance of item()+"/></t>;   
      <f><xsl:value-of select="xs:negativeInteger('-2') instance of node()"/></f>;
      <t><xsl:value-of select="xs:positiveInteger('+2') instance of xs:decimal"/></t>;                                          
    </out> 
  </xsl:template>
  

</xsl:stylesheet>