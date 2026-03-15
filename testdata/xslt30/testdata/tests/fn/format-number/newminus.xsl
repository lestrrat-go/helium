<?xml version="1.0"?> 
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

<!-- This stylesheet is to be imported or included. It defines the 'newminus' format. -->

  <xsl:import href="periodgroup.xsl"/>

  <xsl:decimal-format name="newminus" minus-sign='_' />

</xsl:stylesheet>
