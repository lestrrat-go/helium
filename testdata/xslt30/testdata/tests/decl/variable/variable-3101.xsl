<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Evaluation of params with repeated use of imported stylesheet. -->
  <!-- Elaboration: One of the params passed in to baseTemplate is the result of
    a call to baseSubTemplate. Both are in the imported stylesheet, which could
    be considered a subroutine library. The ...Param0 params aren't actually
    used, but are there to potentially cause trouble. baseTemplate puts out two
    child elements, one from itself and one it got from baseSubTemplate. -->

  <xsl:import href="variable-3101a.xsl"/>

  <xsl:template match="/">
    <out>
      <xsl:call-template name="baseTemplate">
        <xsl:with-param name="baseParam0" select="'baseParam1Data'"/>
        <xsl:with-param name="baseParam1">
          <xsl:call-template name="baseSubTemplate">
            <xsl:with-param name="baseSubParam0" select="'baseSubParam0Data'"/>
          </xsl:call-template>
        </xsl:with-param>
      </xsl:call-template>
    </out>
  </xsl:template>

</xsl:stylesheet>
