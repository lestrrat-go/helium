<?spec xslt#grouping?>
<!-- positional grouping to turn <br/> separators into <line> elements -->
<limerick xsl:version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:for-each-group select="/limerick/node()" group-starting-with="br">
    <line><xsl:copy-of select="current-group()[not(self::br)]"/></line>
  </xsl:for-each-group>
</limerick>