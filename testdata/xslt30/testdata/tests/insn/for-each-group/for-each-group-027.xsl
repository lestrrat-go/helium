<?spec xslt#grouping?>
<!-- grouping used to find all the element names and attribute names in a document -->
<inventory xsl:version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:for-each-group select="//*" group-by="name()">
    <xsl:sort select="name(.)"/>
   * <element name="{name(.)}">
      <xsl:for-each-group select="current-group()/@*"
                          group-by="name()">
        <xsl:sort select="name()"/>
       ! <attribute name="{name(.)}">
          <xsl:for-each select="current-group()"
                        >
            <xsl:sort select="."/>
           - <value><xsl:value-of select="."/></value>
          </xsl:for-each>
        </attribute>
      </xsl:for-each-group>
    </element>
  </xsl:for-each-group>
</inventory>