<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="2.0">

  <!-- Test type-available() - non-availability of non-primitive types with a basic processor  -->


  <xsl:template match="/">
    <out>
      <int>
        <xsl:value-of select="type-available('xs:int')"/>
      </int>
      <short>
        <xsl:value-of select="type-available('xs:short')"/>
      </short>
      <NCName>
        <xsl:value-of select="type-available('xs:NCName')"/>
      </NCName>
      <ID>
        <xsl:value-of select="type-available('xs:ID')"/>
      </ID>
      <NOTATION>
        <xsl:value-of select="type-available('xs:NOTATION')"/>
      </NOTATION>
      <ENTITIES>
        <xsl:value-of select="type-available('xs:ENTITIES')"/>
      </ENTITIES>
      <NMTOKENS>
        <xsl:value-of select="type-available('xs:NMTOKENS')"/>
      </NMTOKENS>
    </out>
  </xsl:template>

</xsl:transform>
