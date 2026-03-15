<xsl:transform
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 version="2.0"
>

<?spec fo#func-count?>
<!-- BUG: Saxon 5.3.2/012 -->
<!-- DESCRIPTION OF BUG: The count() function, when applied to a node-set expressed as a filter expression,
    may give the wrong answer: specifically, it gives the number of nodes before applying the
    predicates. -->

<xsl:variable name="teams" select="//team[not(.=preceding::team)]"/>
<xsl:variable name="matches" select="//match"/>

<xsl:template match="results">
<html><body>
   <h1>Results of Group <xsl:value-of select="@group"/></h1>

   <table cellpadding="5">
      <tr>
        <td>Team</td>
        <td>Played</td>
        <td>Won</td>
        <td>Drawn</td>
        <td>Lost</td>
        <td>For</td>
        <td>Against</td>
     </tr>
   <xsl:for-each select="$teams">
        <xsl:variable name="this" select="."/>
        <xsl:variable name="played" select="count($matches[team=$this])"/>
        <xsl:variable name="won" 
            select="count($matches[if (team=$this) then number(team[.=$this]/@score) &gt; number(team[.!=$this]/@score) else false()])"/>
        <xsl:variable name="lost"
            select="count($matches[if (team=$this) then number(team[.=$this]/@score) &lt; number(team[.!=$this]/@score) else false()])"/>
        <xsl:variable name="drawn"
            select="count($matches[if (team=$this) then number(team[.=$this]/@score) = number(team[.!=$this]/@score) else false()])"/>
        <xsl:variable name="for"
            select="sum($matches/team[.=current()]/@score)"/>
        <xsl:variable name="against"
            select="sum($matches[if (team=$this) then team=current() else false()]/team/@score) - $for"/>

        <tr>
        <td><xsl:value-of select="."/></td>
        <td><xsl:value-of select="$played"/></td>
        <td><xsl:value-of select="$won"/></td>
        <td><xsl:value-of select="$drawn"/></td>
        <td><xsl:value-of select="$lost"/></td>
        <td><xsl:value-of select="$for"/></td>
        <td><xsl:value-of select="$against"/></td>
        </tr>
   </xsl:for-each>
   </table>
</body></html>
</xsl:template>

</xsl:transform>
