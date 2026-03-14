<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- xsl:iterate; reference to local variable with xsl:on-completion -->
  <!-- Was designed to test bug 24179, which was fixed by disallowing variable declarations
       to precede xsl:on-completion. The test now moves xsl:on-completion to
       before the variable declaration, which makes it a trivial varable out-of-scope test. -->

  <xsl:param name="p" as="element()*" select="()"/>

  <xsl:template name="main">
    <out>
      <xsl:iterate select="$p">
        <xsl:param name="count" select="0"/>
        <xsl:on-completion select="$v"/>
        <xsl:variable name="v" select="$count + number(.)"/>
        <xsl:next-iteration>
          <xsl:with-param name="count" select="$count + 1"/>
        </xsl:next-iteration>        
      </xsl:iterate>
    </out>
  </xsl:template>

</xsl:stylesheet>
