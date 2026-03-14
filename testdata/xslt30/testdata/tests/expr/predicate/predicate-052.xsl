<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
version="2.0">

    <!-- two calls on current() in a single predicate - bug in Saxon 8.4 -->
    <xsl:output method="xml"/>

    <xsl:template match="/">
    <out>
      <xsl:variable name="a">
        <x xAtt="1"/>
        <x xAtt="2"/>
        <x xAtt="3"/>
      </xsl:variable>
     
      <xsl:variable name="b">
        <y yAtt="3"/>
      </xsl:variable>
     
      <xsl:for-each select="$b//y">

        <xsl:if test="$a//x[number(@xAtt) le number(current()/@yAtt)]">
          <a>ok1</a>
        </xsl:if>
        <xsl:if test="$a//x[number(@xAtt) le number(current()/@yAtt) and 
number(@xAtt) ge number(current()/@yAtt)]">
          <a>ok2</a>
        </xsl:if>
      </xsl:for-each>
    </out>
    </xsl:template>

</xsl:stylesheet>
