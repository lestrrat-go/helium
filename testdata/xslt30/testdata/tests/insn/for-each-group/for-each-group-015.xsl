<?spec xslt#grouping?>
<!-- current-grouping-key when a positional for-each-group occurs within a value-based for-each-group -->
<!-- see bug 18333 -->
<out xsl:version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:for-each-group select="cities/*" group-by="node-name(.)">
    <g>
      <xsl:for-each-group select="current-group()" group-starting-with="*[@country='france']">
        <h key="{current-grouping-key()}"/>
      </xsl:for-each-group>
    </g>
  </xsl:for-each-group>
</out>
