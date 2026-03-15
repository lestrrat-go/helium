<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">

  <xsl:variable name="RUN" static="yes" select="true()"/>

  <!-- Test of xsl:source-document with distinct-values() applied to an attribute -->

  <xsl:template name="d-001" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
      <out>
        <xsl:value-of select="count(distinct-values(account/transaction/@date))"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test of xsl:source-document with distinct-values() - empty input -->

  <xsl:template name="d-002" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
      <out>
        <xsl:value-of select="count(distinct-values(account/transaction/@dummy-date))"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test of xsl:source-document with distinct-values() - numeric values -->

  <xsl:template name="d-003" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
      <out>
        <xsl:value-of select="count(distinct-values(account/transaction/number(@value)))"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test of xsl:source-document with distinct-values() - handling of NaN -->

  <xsl:template name="d-004" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
      <out>
        <xsl:value-of select="count(distinct-values(account/transaction/number(concat('-', @value))))"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test of xsl:source-document with distinct-values() - applied to an element -->

  <xsl:template name="d-005" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="distinct-values(outermost(//LANGUAGE))"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test of xsl:source-document with distinct-values() - with collation specified -->

  <xsl:template name="d-006" use-when="$RUN">
    <xsl:param name="c" select="'http://www.w3.org/2005/xpath-functions/collation/codepoint'"/>
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="distinct-values(outermost(//LANGUAGE), $c)"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test of xsl:source-document with distinct-values() - with unknown collation specified -->

  <xsl:template name="d-007" use-when="$RUN">
    <xsl:param name="c" select="'http://www.w3.org/2005/xpath-functions/collation/unknown'"/>
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="distinct-values(outermost(//LANGUAGE), $c)"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test of xsl:source-document with distinct-values() - with unknown collation specified, error is caught -->

  <xsl:template name="d-008" use-when="$RUN">
    <xsl:param name="c" select="'http://www.w3.org/2005/xpath-functions/collation/unknown'"/>
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:try>
          <xsl:value-of select="distinct-values(outermost(//LANGUAGE), $c)"/>
          <xsl:catch errors="*:FOCH0002" select="'caught'"/>
        </xsl:try>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test of xsl:source-document with distinct-values() - with collation from consuming argument -->

  <xsl:template name="d-009" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/special.xml">
      <out>
        <xsl:value-of select="distinct-values(('a', 'b', 'a'), /special/codepointCollation)"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test of xsl:source-document with distinct-values() - with unknown collation from consuming argument -->

  <xsl:template name="d-010" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/special.xml">
      <out>
        <xsl:value-of select="distinct-values(('a', 'b', 'a'), /special/unknownCollation)"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test of xsl:source-document with distinct-values() - with unknown collation from consuming argument, error is caught -->

  <xsl:template name="d-011" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/special.xml">
      <out>
        <xsl:try>
          <xsl:value-of select="distinct-values(('a', 'b', 'a'), /special/unknownCollation)"/>
          <xsl:catch errors="*:FOCH0002" select="'caught'"/>
        </xsl:try>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test that double NaN and float NaN are not distinct -->

  <xsl:template name="d-012" use-when="$RUN">
    <xsl:variable name="NaN" select="xs:double('NaN')"/>
    <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
      <out>
        <xsl:value-of
          select="count(distinct-values(account/transaction/number(@value) ! (if (. lt 0) then $NaN else xs:float($NaN))))"/>
      </out>
    </xsl:source-document>
  </xsl:template>

</xsl:stylesheet>
