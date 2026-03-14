<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:m ="urn:non-null-namespace">

<?spec xslt#current-function?>
<!-- BUG: Saxon 5.3.2/027 -->
<!-- DESCRIPTION:
    When the current() function is used within the select expression of xsl:sort,
    it gives the wrong answer (the relevant node will be the context node, but not the
    current node). This results in an incorrect sort sequence. -->

<m:month name="Jan" value="1" />
<m:month name="Feb" value="2" />
<m:month name="Mar" value="3" />
<m:month name="Apr" value="4" />
<m:month name="May" value="5" />
<m:month name="Jun" value="6" />
<m:month name="Jul" value="7" />
<m:month name="Aug" value="8" />
<m:month name="Sep" value="9" />
<m:month name="Oct" value="10" />
<m:month name="Nov" value="11" />
<m:month name="Dec" value="12" />

<xsl:template match="list">
<out>
   <xsl:apply-templates select="*">
      <xsl:sort select="document('')//m:month[@name=current()/@month]/@value" 
                data-type="number"/>
   </xsl:apply-templates>
</out>
</xsl:template>

<xsl:template match="item">
* <item month="{@month}"/>
</xsl:template>

</xsl:stylesheet>

