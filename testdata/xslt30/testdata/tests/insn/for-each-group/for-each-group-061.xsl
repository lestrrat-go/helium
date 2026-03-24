<?spec xslt#grouping?>
<xsl:transform version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- Demonstrate group-by using times in different time zones -->


  <xsl:template match="/">
    <out>
      <xsl:for-each-group select="//city" group-by="xs:dateTime(@date)">
        <group date="{@date}">
          <xsl:copy-of select="current-group()"/>
        </group>
      </xsl:for-each-group>
    </out>
  </xsl:template>

</xsl:transform>
