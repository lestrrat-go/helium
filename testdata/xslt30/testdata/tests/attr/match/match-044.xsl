<?xml version="1.0"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  version="2.0"

>
<?spec xslt#applying-templates?>

<xsl:param name="countryCode"/>

<xsl:template match="attributes">
  <s><xsl:apply-templates/></s>
</xsl:template>

<xsl:template match="attribute">
  <pling/>
</xsl:template>



</xsl:transform>
