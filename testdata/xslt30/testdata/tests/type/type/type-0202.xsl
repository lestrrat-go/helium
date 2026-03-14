<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test "instance of" on attributes. -->
  <?spec xpath#id-instance-of?>
  <?spec xpath#id-attribute-test?>
  <?spec xpath#id-sequencetype-syntax?>

  <xsl:template match="/">
    <out>;
      <t><xsl:value-of select="//@* instance of attribute() *"/></t>;
      <f><xsl:value-of select="(//@* | //*) instance of attribute() *"/></f>;
      <t><xsl:value-of select="//@* instance of node() *"/></t>;
      <t><xsl:value-of select="//@* instance of item() +"/></t>;
      <f><xsl:value-of select="/a/b/@c instance of attribute()"/></f>;
      <t><xsl:value-of select="/a/b/@c instance of attribute() *"/></t>;      
    </out>
  </xsl:template>

</xsl:stylesheet>