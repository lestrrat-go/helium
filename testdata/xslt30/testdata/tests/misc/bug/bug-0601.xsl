<?xml version="1.0" ?>

<!-- BUG: Saxon 5.3/005 and 5.3/008 -->
<!-- DESCRIPTION: 5.3/005: If xml:namespace-alias is used in an included or imported stylesheet module,
    it is ignored. 5.3/008: When xsl:copy-of is used to copy a result tree fragment to the final tree,
    no attempt is made to remove redundant namespace declarations. -->

<xsl:stylesheet
   version="2.0"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:axsl="http://www.w3.org/1999/XSL/TransformAlias">

<xsl:import href="bug-0601a.xsl"/>

<xsl:template name="process-prolog">
   <axsl:output method="text" />
</xsl:template>

<xsl:template name="process-root">
   <xsl:param name="title" />
   <xsl:param name="contents" />
   <xsl:value-of select="$title" />
   <xsl:text>&#10;</xsl:text>
   <xsl:copy-of select="$contents" />
</xsl:template>

<!-- use default rule for process-pattern: ignore name and see -->
<!-- use default rule for process-name: output name -->
<!-- use default rule for process-assert and process-report:
     call process-message -->

<xsl:template name="process-message">
   <xsl:param name="pattern" />
   <xsl:param name="role" />
   <xsl:text>In pattern </xsl:text>
   <xsl:value-of select="$pattern" />
   <xsl:if test="$role">
      <xsl:text> (</xsl:text>
      <xsl:value-of select="$role" />
      <xsl:text>)</xsl:text>
   </xsl:if>:
   <xsl:apply-templates mode="text" />
   <xsl:text>&#10;</xsl:text>
</xsl:template>

</xsl:stylesheet>
