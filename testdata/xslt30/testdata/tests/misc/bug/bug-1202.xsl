<out xsl:version="2.0"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xslt#disable-output-escaping?>
<!-- BUG: Saxon 5.3.1/009 -->
<!-- DESCRIPTION: disable-output-escaping="yes" has no effect when writing to a result tree
    fragment. -->

    <!-- test whether disable-output-escaping is "sticky" -->

    <a><xsl:text>Enable escaping here &lt;fred/&gt;</xsl:text></a>;
    <b><xsl:text disable-output-escaping="yes">Disable escaping here &lt;fred/&gt;</xsl:text></b>;
    <c><xsl:text disable-output-escaping="no">Enable escaping here &lt;fred/&gt;</xsl:text></c>;

    <xsl:variable name="rtf">
    <p><xsl:text>Enable escaping here &lt;fred/&gt;</xsl:text></p>;
    <q><xsl:text disable-output-escaping="yes">Disable escaping here &lt;fred/&gt;</xsl:text></q>;
    <r><xsl:text disable-output-escaping="no">Enable escaping here &lt;fred/&gt;</xsl:text></r>;
    </xsl:variable>
    <xsl:copy-of select="$rtf"/>
</out>

