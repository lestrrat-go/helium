<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

<!-- 
     Test stylesheet for testing character classes within regular expressions.
     
     The character class to be tested is passed in as a parameter. The stylesheet
     generates output showing all the characters in the class, and in its complement.
     
     Alternative results are acceptable depending on which version of Unicode the processor
     supports. Because the test suite catalog format does not readily allow for alternative
     results, the stylesheet is self-checking: it tests the result against reference results
     in a number of known subdirectories: if the results are equal to any of the reference results,
     it returns <true/>, otherwise it returns the actual results obtained.
     
     Author: Michael Kay
     Date: 14 January 2010
-->

<xsl:param name="class" as="xs:string" required="yes"/>

<xsl:variable name="allLegalChars"  as="xs:integer*" 
              select="(9 to 10, 13, 32 to 55295, 57344 to 65533, 65536 to 1114111 )"/>
              
<xsl:variable name="omnibus" as="xs:string"
              select="codepoints-to-string($allLegalChars)"/>
              
<xsl:param name="result-dirs" select="'3.1', '5.2', '6.0'"/>
  
<!-- save-results allows collection of results for a new Unicode version -->  
<xsl:param name="save-results" select="false()"/>  

<xsl:variable name="class-regex" select="concat('\p{', $class, '}+')"/>
<xsl:variable name="complement-regex" select="concat('\P{', $class, '}+')"/>

<xsl:strip-space elements="*"/>

<xsl:output encoding="us-ascii"/>

<xsl:variable name="results">
<out>
  <ranges category="{$class}">
     <xsl:analyze-string select="$omnibus" regex="{$class-regex}">
       <xsl:matching-substring>
         <xsl:variable name="codes" select="string-to-codepoints(.)"/>
         <match from="{$codes[1]}" to="{$codes[last()]}"/>
       </xsl:matching-substring>
       <xsl:non-matching-substring/>
     </xsl:analyze-string>
  </ranges>
  <ranges complement="{$class}">
     <xsl:analyze-string select="$omnibus" regex="{$complement-regex}">
       <xsl:matching-substring>
         <xsl:variable name="codes" select="string-to-codepoints(.)"/>
         <match from="{$codes[1]}" to="{$codes[last()]}"/>
       </xsl:matching-substring>
       <xsl:non-matching-substring/>
     </xsl:analyze-string>
  </ranges>
</out>
</xsl:variable>

<xsl:template name="go">
  <xsl:variable name="comparisons" as="element(true)*">
    <xsl:for-each select="$result-dirs">
      <xsl:variable name="result-doc" select="concat(., '/', $class, '.xml')"/>
      <xsl:if test="doc-available($result-doc)">
        <xsl:if test="deep-equal($results, doc($result-doc))">
          <true/>
        </xsl:if>
      </xsl:if>
    </xsl:for-each>
  </xsl:variable>
  <xsl:choose>
    <xsl:when test="exists($comparisons)">
      <true/>
    </xsl:when>
    <xsl:otherwise>
      <xsl:copy-of select="$results"/>
    </xsl:otherwise>
  </xsl:choose>
  <xsl:if test="$save-results">
    <xsl:result-document href="{$result-dirs[last()]}R/{$class}.xml" method="xml" indent="yes">
       <xsl:message>Sending results to <xsl:value-of select="$class"/>.xml</xsl:message>
       <xsl:copy-of select="$results"/>
    </xsl:result-document>
  </xsl:if>
</xsl:template>

</xsl:transform>  