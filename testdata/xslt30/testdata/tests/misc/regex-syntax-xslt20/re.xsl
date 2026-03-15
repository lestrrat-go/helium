<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

<!-- 
     Test stylesheet for testing regular expressions.
     
     The regular expression and the strings to be tested are passed as parameters
     from the test catalog.
     
     Author: Michael Kay
     Date: 11 November 2007
     
     The test data in the catalog that is used with this stylesheet is derived from the
     Microsoft-submitted regular expression tests in the XML Schema test suite. The classification
     of results as invalid, matching, or matching was done by Saxonica.
     
     Note that because the supplied regex uses schema syntax, we have to add anchors to reproduce
     the schema semantics.
-->

<xsl:param name="regex" as="xs:string" required="yes"/>
<xsl:param name="match" as="xs:string" select="'----'"/>
<xsl:param name="nonmatch" as="xs:string" select="''"/>
<xsl:param name="delimiter" as="xs:string" select="','"/>

<xsl:variable name="anchored-regex" select="concat('^(', $regex, ')$')"/>

<xsl:output encoding="us-ascii" indent="yes"/>

<xsl:template name="go">
<xsl:choose>
  <xsl:when test="(every $s in tokenize($match, $delimiter) satisfies matches($s, $anchored-regex))
           and (every $s in tokenize($nonmatch, $delimiter) satisfies not(matches($s, $anchored-regex)))">
    <true/>
  </xsl:when>
  <xsl:otherwise>
    <!-- output here is for diagnostics -->
    <regex value="{$regex}">
      <xsl:for-each select="tokenize($match, $delimiter)">
        <xsl:if test="not(matches(., $anchored-regex))">
          <string value="{.}" cp="{string-to-codepoints(.)}" expected-match="true"  actual-match="false"/>
        </xsl:if>  
      </xsl:for-each> 
      <xsl:for-each select="tokenize($nonmatch, $delimiter)">
        <xsl:if test="matches(., $anchored-regex)">
          <string value="{.}" cp="{string-to-codepoints(.)}" expected-match="false" actual-match="true"/>
        </xsl:if>
      </xsl:for-each>
    </regex>
  </xsl:otherwise>
</xsl:choose>
</xsl:template>

</xsl:transform>  