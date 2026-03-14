<?xml version="1.0" encoding="UTF-8"?>
<!--It is a type error if the sequence
                     returned by the select expression of
                           xsl:apply-templates
                      contains an item that is not a node.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:apply-templates (select att must be a node()*)-->
<?spec xslt#errors?><?error XTSE0520?>

  <xsl:template match="doc">
      <out>
         <xsl:apply-templates select="3">
            <xsl:sort select="."/>
         </xsl:apply-templates> 
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>

</xsl:stylesheet>
