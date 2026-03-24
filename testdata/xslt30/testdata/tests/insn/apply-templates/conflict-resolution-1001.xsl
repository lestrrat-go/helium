<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

    <?spec xslt#defining-template-rules?>
    <!-- tests resolution of Saxon bug 576632 -->

    <!-- parametre externe "type" (tous, parent, enfant) -->
    <xsl:param name="type"/>

    <!-- regle-modele pour la racine -->
    <xsl:template match="/">
        <planche>
            <images>
                <xsl:apply-templates select="planche/images/*[@type=$type]"/>
            </images>
            <dialogues>
                <xsl:apply-templates select="planche/dialogues/*[@type=$type]"/>
            </dialogues>
        </planche>
    </xsl:template>

    <!-- regle-modele pour les noeuds fils de images fils de planche -->
    <xsl:template match="planche/images/*[@type=$type]"> truc <xsl:copy-of select="."/>
    </xsl:template>

    <!-- regle-modele pour les noeuds fils de dialogues fils de planche -->
    <xsl:template match="planche/dialogues/*[@type=$type]"> toto <xsl:copy-of select="."/>
    </xsl:template>

</xsl:stylesheet>
