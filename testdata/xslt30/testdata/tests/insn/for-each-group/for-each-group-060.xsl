<?spec xslt#grouping?>
<xsl:transform version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- in 3.0, current-group() is not visible within the AVTs of xsl:sort -->

  <xsl:template match="/">
    <err>
      <xsl:for-each-group select="'a', 1 to 10, 'b', 11 to 20"
        group-adjacent=". instance of xs:string">
        <xsl:sort select="count(current-group())" order="{if (current-group()[1]) then 'ascending' else 'descending'}"/>
        <a>
          <xsl:value-of select="current-group()"/>
        </a>
      </xsl:for-each-group>
    </err>
  </xsl:template>

</xsl:transform>
