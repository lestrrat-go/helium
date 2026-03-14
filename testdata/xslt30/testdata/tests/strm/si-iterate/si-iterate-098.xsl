<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:ls="http://loan.shark.com/"
  exclude-result-prefixes="ls xs" xpath-default-namespace="http://loan.shark.com/" version="3.0">


  <!-- within xsl:stream, xsl:iterate over descendants -->


  <xsl:output method="xml" indent="no" encoding="UTF-8"/>

  <xsl:param name="sep" select="'|'" as="xs:string"/>

  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:iterate
          select="outermost(descendant::element()[name(.)=('Property','PropertyValuation')])">
          <n>
            <xsl:value-of select="name(.)"/>
          </n>
        </xsl:iterate>
      </xsl:source-document>
    </out>
  </xsl:template>


</xsl:transform>
